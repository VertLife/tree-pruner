from google.appengine.ext import vendor
from google.appengine.api import namespace_manager

vendor.add('lib')

# # Called only if the current namespace is not set.
# def namespace_manager_default_namespace_for_request():
#   # raise
#   return namespace_manager.google_apps_namespace()
