// Copyright 2018 Google Cloud Platform Proxy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

#pragma once

#include <string>

#include "common/common/logger.h"
#include "envoy/http/header_map.h"
#include "src/api_proxy/service_control/request_builder.h"
#include "src/envoy/http/service_control/filter_config.h"
#include "src/envoy/http/service_control/handler.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

// The request handler to call Check and Report
class ServiceControlHandlerImpl : public Logger::Loggable<Logger::Id::filter>,
                                  public ServiceControlHandler {
 public:
  ServiceControlHandlerImpl(const Http::HeaderMap& headers,
                            const StreamInfo::StreamInfo& stream_info,
                            const ServiceControlFilterConfig& config);
  virtual ~ServiceControlHandlerImpl();

  void callCheck(Http::HeaderMap& headers, CheckDoneCallback& callback);

  void callReport(const Http::HeaderMap* request_headers,
                  const Http::HeaderMap* response_headers,
                  const Http::HeaderMap* response_trailers);

 private:
  // Helper functions to extract API key.
  bool extractAPIKeyFromQuery(const Http::HeaderMap& headers,
                              const std::string& query);
  bool extractAPIKeyFromHeader(const Http::HeaderMap& headers,
                               const std::string& header);
  bool extractAPIKeyFromCookie(const Http::HeaderMap& headers,
                               const std::string& cookie);
  bool extractAPIKey(
      const Http::HeaderMap& headers,
      const ::google::protobuf::RepeatedPtrField<
          ::google::api::envoy::http::service_control::APIKeyLocation>&
          locations);
  void fillLoggedHeader(
      const Http::HeaderMap* headers,
      const ::google::protobuf::RepeatedPtrField<::std::string>& log_headers,
      std::string& info_log_header_field);
  void fillOperationInfo(
      ::google::api_proxy::service_control::OperationInfo& info);
  void fillGCPInfo(
      ::google::api_proxy::service_control::ReportRequestInfo& info);

  bool isConfigured() const { return require_ctx_ != nullptr; }

  bool isCheckRequired() const {
    return !require_ctx_->config().api_key().allow_without_api_key();
  }

  bool hasApiKey() const { return !api_key_.empty(); }

  void onCheckResponse(
      Http::HeaderMap& headers, const ::google::protobuf::util::Status& status,
      const ::google::api_proxy::service_control::CheckResponseInfo&
          response_info);

  // The filer config.
  const ServiceControlFilterConfig& config_;

  // The metadata for the request
  const StreamInfo::StreamInfo& stream_info_;

  // The matched requirement
  const RequirementContext* require_ctx_{};

  // cached parsed query parameters
  bool params_parsed_{false};
  Http::Utility::QueryParams parsed_params_;

  std::string path_;
  std::string http_method_;
  std::string uuid_;
  std::string api_key_;

  CheckDoneCallback* check_callback_{};
  ::google::api_proxy::service_control::CheckResponseInfo check_response_info_;
  ::google::protobuf::util::Status check_status_;

  // This flag is used to mark if the request is aborted before the check
  // callback is returned.
  std::shared_ptr<bool> aborted_;
  uint64_t request_header_size_;
};

class ServiceControlHandlerFactoryImpl : public ServiceControlHandlerFactory {
 public:
  ServiceControlHandlerFactoryImpl() {}
  ~ServiceControlHandlerFactoryImpl() {}

  ServiceControlHandlerPtr createHandler(
      const Http::HeaderMap& headers, const StreamInfo::StreamInfo& stream_info,
      const ServiceControlFilterConfig& config) const override {
    return std::make_unique<ServiceControlHandlerImpl>(headers, stream_info,
                                                       config);
  }
};

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy